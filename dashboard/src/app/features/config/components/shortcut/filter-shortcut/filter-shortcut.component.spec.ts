import {ComponentFixture, TestBed} from '@angular/core/testing';

import {FilterShortcutComponent} from './filter-shortcut.component';
import {ConfigObject, Kind} from '../../../../../shared/models';
import {provideCoreTesting} from '../../../../../core/core.testing.module';

describe('FilterShortcutComponent', () => {
  let component: FilterShortcutComponent;
  let fixture: ComponentFixture<FilterShortcutComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [FilterShortcutComponent],
      providers: [
        ...provideCoreTesting
      ],
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(FilterShortcutComponent);
    component = fixture.componentInstance;
    component.configObject = new ConfigObject();
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should create', async () => {
    component.configObject = new ConfigObject({kind: Kind.CRAWLENTITY});
    await fixture.whenStable();
    expect(component).toBeTruthy();
  });

  it('should create with SEED config', async () => {
    component.configObject = new ConfigObject({kind: Kind.SEED});
    await fixture.whenStable();
    expect(component).toBeTruthy();
  });

  it('should create with CRAWLJOB config', async () => {
    component.configObject = new ConfigObject({kind: Kind.CRAWLJOB});
    await fixture.whenStable();
    expect(component).toBeTruthy();
  });
});
