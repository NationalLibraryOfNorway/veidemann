import {ComponentFixture, TestBed} from '@angular/core/testing';
import {PageLogStatusComponent} from './page-log-status.component';
import {provideCoreTesting} from '../../../../core/core.testing.module';
import {PageLog, Resource} from '../../../../shared/models';
import {provideRouter} from '@angular/router';

describe('PageLogStatusComponent', () => {
  let component: PageLogStatusComponent;
  let fixture: ComponentFixture<PageLogStatusComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [PageLogStatusComponent],
      providers: [...provideCoreTesting, provideRouter([])]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(PageLogStatusComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('shows tab counts, filters resources, and preserves malformed outlinks', () => {
    const pageLog = new PageLog({
      resource: [
        new Resource({uri: 'https://example.org/image.png', mimeType: 'image/png'}),
        new Resource({uri: 'https://example.org/app.js', mimeType: 'text/javascript'}),
      ],
      outlink: ['https://example.org/path', 'not a uri'],
    });
    fixture.componentRef.setInput('pageLog', pageLog);
    fixture.detectChanges();

    expect(fixture.nativeElement.textContent).toContain('Resources (2)');
    expect(fixture.nativeElement.textContent).toContain('Outlinks (2)');
    component.applyResourceFilter('IMAGE');
    expect(component.filteredResources.map(resource => resource.uri)).toEqual(['https://example.org/image.png']);
    component.applyOutlinkFilter('not a uri');
    expect(component.filteredOutlinks).toEqual([expect.objectContaining({raw: 'not a uri', href: null})]);
  });

  it('projects helpers into the card header and renders the Outlinks external-link icon as list metadata', async () => {
    fixture.componentRef.setInput('pageLog', new PageLog({
      uri: 'https://example.org',
      outlink: ['https://outlink.example/path'],
    }));
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('mat-card-header.card-header-with-helpers')).not.toBeNull();
    const tabs = fixture.nativeElement.querySelectorAll('[role="tab"]') as NodeListOf<HTMLElement>;
    tabs[1].click();
    await fixture.whenStable();
    fixture.detectChanges();
    expect(fixture.nativeElement.querySelector('mat-nav-list a[target="_blank"] span mat-icon')?.textContent)
      .toContain('open_in_new');
  });

  it('opens a metadata dialog with resource error details', () => {
    const dialog = component['dialog'];
    const open = vi.spyOn(dialog, 'open').mockReturnValue({} as never);
    const resource = new Resource({uri: 'https://example.org', statusCode: 500});
    resource.error.code = 7;
    resource.error.msg = 'Failed';

    component.showMetadata(resource);

    expect(open).toHaveBeenCalledWith(expect.anything(), expect.objectContaining({
      data: expect.arrayContaining([expect.objectContaining({label: 'Error', value: '7: Failed'})]),
    }));
  });
});
