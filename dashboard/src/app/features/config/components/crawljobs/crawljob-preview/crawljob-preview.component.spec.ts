import {ComponentFixture, TestBed} from '@angular/core/testing';
import {CrawljobPreviewComponent} from './crawljob-preview.component';
import {ConfigurationsModule} from '../../../configurations.module';
import {CoreTestingModule} from '../../../../core/core.testing.module';
import {ConfigObject, Kind} from '../../../../shared/models';
import {AuthService} from '../../../../core';
import {CommonsModule} from '../../../../commons';

describe('CrawljobPreviewComponent', () => {
  let component: CrawljobPreviewComponent;
  let fixture: ComponentFixture<CrawljobPreviewComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [
        CommonsModule,
        ConfigurationsModule,
        CoreTestingModule.forRoot()
      ],
      declarations: [CrawljobPreviewComponent],
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(CrawljobPreviewComponent);
    component = fixture.componentInstance;
    component.configObject = new ConfigObject({kind: Kind.CRAWLJOB});
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
